function shotgunhit(mc)
{
   switch(Math.floor(asdf / 1000))
   {
      case 1:
         _root.pgsdata[0][4] += 1;
         break;
      case 2:
         _root.pgsdata[1][4] += 1;
         break;
      case 3:
         _root.pgsdata[2][4] += 1;
         break;
      case 4:
         _root.pgsdata[3][4] += 1;
   }
   mc.hitbynade = false;
}
function tagplayer(mc)
{
   switch(Math.floor(asdf / 1000))
   {
      case 1:
         mc.gothitby = _root.p1name;
         break;
      case 2:
         mc.gothitby = _root.p2name;
         break;
      case 3:
         mc.gothitby = _root.p3name;
         break;
      case 4:
         mc.gothitby = _root.p4name;
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
speed = 25 + random(6) - 3;
facing = 1;
vx = Math.cos(_rotation * 3.141592653589793 / 180) * speed;
vy = Math.sin(_rotation * 3.141592653589793 / 180) * speed;
if(_rotation > 90 || _rotation < -90)
{
   facing = -1;
}
time = 1;
trail._alpha = 0;
freepass = false;
firepower = Math.abs(asdf % 1000);
time = 0;
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      if(!removethis)
      {
         _X = _X + vx;
         _Y = _Y + vy;
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
                     shotgun = _root.activeplayers[i].hitnumber;
                     _root.activeplayers[i].hittimer = 0;
                     qwer = _root.activeplayers[i].hitnumber;
                     _root.CP("fx_blood",_X,_Y,0,0);
                     _root.CP("fx_bloodstain",_X + speed,_Y,0,0);
                     if(shotgun == 6 && firepower >= 7)
                     {
                        _root.CP("fx_combo",_X + speed,_Y - 50,0,-4);
                     }
                     else if(shotgun == 1)
                     {
                        _root.hitsound();
                        _root.CP("fx_combo",_X + speed,_Y - 50,0,qwer);
                        shotgunhit(_root.activeplayers[i]);
                     }
                     tagplayer(_root.activeplayers[i]);
                     deletethis();
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
         if(time == 5)
         {
            trail._alpha = 100;
         }
         if(firepower >= 7)
         {
            if(time > 6)
            {
               _alpha = _alpha - 25;
            }
            if(time > 9)
            {
               deletethis();
            }
         }
         else
         {
            if(time > 8)
            {
               _alpha = _alpha - 25;
            }
            if(time > 11)
            {
               deletethis();
            }
         }
         if(trail._width > 20)
         {
            trail._width -= 20;
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
