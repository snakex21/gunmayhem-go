function tagplayer(mc)
{
   switch(Math.floor(asdf / 1000))
   {
      case 1:
         mc.gothitby = _root.p1name;
         _root.pgsdata[0][4] += 1;
         break;
      case 2:
         mc.gothitby = _root.p2name;
         _root.pgsdata[1][4] += 1;
         break;
      case 3:
         mc.gothitby = _root.p3name;
         _root.pgsdata[2][4] += 1;
         break;
      case 4:
         mc.gothitby = _root.p4name;
         _root.pgsdata[3][4] += 1;
   }
   mc.hitbynade = false;
}
function dothehittest()
{
   if(!freepass)
   {
      if(_root.ground.hitTest(_X,_Y,true))
      {
         _X = _X - speed;
         i = 1;
         while(i <= 5)
         {
            if(_root.ground.hitTest(_X + i * (speed / 5),_Y,true))
            {
               _X = _X + speed / 5 * (i - 1);
               break;
            }
            i++;
         }
         i = 0;
         while(i < 3)
         {
            _root.CP("fx_spark",_X,_Y,0);
            _root.CP("fx_debris",_X,_Y,0);
            i++;
         }
         removethis = true;
      }
   }
}
function deletethis()
{
   removeMovieClip(this);
   delete this.onEnterFrame;
}
speed = 25;
facing = 1;
if(_rotation != 0)
{
   speed *= -1;
   facing = -1;
}
time = 1;
freepass = false;
firepower = Math.abs(asdf % 1000);
if(firepower > 60)
{
   trail.gotoAndStop(2);
}
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      if(!removethis)
      {
         fx._xscale = random(50) + 50;
         fx._yscale = fx._xscale;
         _X = _X + speed;
         _root.CP("fx_instatrail",_X,_Y,-90 + facing * 90,0);
         if(!removethis)
         {
            i = 0;
            while(i < _root.activeplayers.length)
            {
               if(_root.activeplayers[i].teamnumber != asdf2)
               {
                  if(_root.activeplayers[i].PLAYERNUMBER != Math.floor(asdf / 1000) && _root.activeplayers[i].frame.hitTest(_X,_Y,true))
                  {
                     _root.activeplayers[i].vx += firepower * facing * _root.activeplayers[i].damagemulti;
                     _root.activeplayers[i].hitnumber += 1;
                     _root.activeplayers[i].hittimer = 0;
                     qwer = _root.activeplayers[i].hitnumber;
                     _root.CP("fx_combo",_X + speed,_Y - 50,0,qwer);
                     gotkilled();
                     tagplayer(_root.activeplayers[i]);
                     _root.activeplayers[i].instagib.play();
                     deletethis();
                  }
                  if(_root.activeplayers[i].AI)
                  {
                     if(_root.activeplayers[i].frame.hitTest(_X + speed * 4,_Y,true) || _root.activeplayers[i].frame.hitTest(_X + speed * 6,_Y,true))
                     {
                        if(_root.activeplayers[i].jumpnum == 2)
                        {
                           _root.activeplayers[i].KEYUP = true;
                        }
                        if(firepower >= 10 && _root.activeplayers[i].vy > -8 && _root.activeplayers[i].jumpnum == 1)
                        {
                           _root.activeplayers[i].KEYUP = true;
                        }
                     }
                  }
               }
               i++;
            }
         }
         if(_X < -600 || _X > 1500)
         {
            deletethis();
         }
         if(trail._alpha < 100)
         {
            trail._alpha += 33;
         }
         if(trail._width < 100)
         {
            trail._width += 10;
         }
      }
      else
      {
         _alpha = _alpha - 25;
         if(_alpha <= 1)
         {
            deletethis();
         }
      }
      if(_root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
