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
trail._width = 20;
trail._alpha = 0;
freepass = false;
firepower = Math.abs(asdf % 1000);
if(firepower > 50)
{
   trail.gotoAndStop(2);
}
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      if(!removethis)
      {
         _X = _X + speed;
         if(!removethis)
         {
            i = 0;
            while(i < _root.activeplayers.length)
            {
               if(_root.activeplayers[i].teamnumber != asdf2)
               {
                  if(_root.activeplayers[i].PLAYERNUMBER != Math.floor(asdf / 1000) && _root.activeplayers[i].frame.hitTest(_X,_Y,true))
                  {
                     if(_root.activeplayers[i].shieldtime == 0)
                     {
                        _root.activeplayers[i].vx += firepower * facing * _root.activeplayers[i].damagemulti;
                     }
                     else
                     {
                        _root.activeplayers[i].vx += firepower * facing * 0.33;
                        _root.activeplayers[i].shield.gotoAndPlay(23);
                     }
                     _root.activeplayers[i].hitnumber += 1;
                     _root.activeplayers[i].hittimer = 0;
                     qwer = _root.activeplayers[i].hitnumber;
                     _root.hitsound();
                     j = 0;
                     while(j < 3)
                     {
                        _root.CP("fx_blood",_X,_Y,0,0);
                        if(_root.activeplayers[i].iszombie && j == 1)
                        {
                           break;
                        }
                        j++;
                     }
                     _root.CP("fx_bloodstain",_X + speed,_Y,0,0);
                     if(firepower >= 40)
                     {
                        _root.CP("fx_combo",_X + speed,_Y - 50,0,-1);
                     }
                     else
                     {
                        _root.CP("fx_combo",_X + speed,_Y - 50,0,qwer);
                     }
                     tagplayer(_root.activeplayers[i]);
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
                        if(firepower >= 40 && _root.activeplayers[i].vy > -8 && _root.activeplayers[i].jumpnum == 1)
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
         time += 1;
         if(time >= 3 && trail._alpha < 100)
         {
            trail._alpha += 50;
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
